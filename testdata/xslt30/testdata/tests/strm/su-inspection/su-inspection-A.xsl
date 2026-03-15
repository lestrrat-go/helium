<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://www.w3.org/xslt30tests/functions" exclude-result-prefixes="map xs err f">

  <xsl:variable name="RUN" select="false()" static="yes"/>
  <xsl:strip-space elements="*"/>


  <!-- Test arity=1 inspection function -->

  <xsl:function name="f:depth-001" as="xs:integer" streamability="inspection">
    <xsl:param name="input" as="node()"/>
    <xsl:sequence select="count($input/ancestor-or-self::*)"/>
  </xsl:function>

  <xsl:template name="t-001" use-when="true() or $RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM) ! f:depth-001(.)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test arity=1 recursive inspection function -->
  
  <xsl:function name="f:depth-002" as="xs:integer" streamability="inspection">
    <xsl:param name="input" as="node()"/>
    <xsl:sequence select="if ($input/.. instance of document-node()) then 1 else f:depth-002($input/..) + 1"/>
  </xsl:function>
  
  <xsl:template name="t-002" use-when="true() or $RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM) ! f:depth-002(.)"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- Test arity=2 inspection function -->

  <xsl:function name="f:get-attribute-value-003" as="xs:string"  streamability="inspection" use-when="true() or $RUN">
    <xsl:param name="element" as="node()"/>
    <xsl:param name="attribute-name" as="xs:string"/>
    <xsl:sequence select="string($element/@*[local-name() = $attribute-name])"/>
  </xsl:function>

  <xsl:template name="t-003" use-when="true() or $RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="/BOOKLIST/BOOKS/ITEM/WEIGHT ! f:get-attribute-value-003(., 'UNIT')"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- Test arity=2 inspection function with recursion -->
  
  <xsl:function name="f:get-inherited-attribute-value-004"  streamability="inspection" as="xs:string?" use-when="true() or $RUN">
    <xsl:param name="element" as="node()?"/>
    <xsl:param name="attribute-name" as="xs:string"/>
    <xsl:sequence select="
      if (empty($element)) then ()
      else if (exists($element/@*[local-name() = $attribute-name])) 
      then string($element/@*[local-name() = $attribute-name])
      else f:get-inherited-attribute-value-004($element/.., $attribute-name)"/>
  </xsl:function>
  
  <xsl:template name="t-004" use-when="true() or $RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="/BOOKLIST/BOOKS/ITEM/WEIGHT ! f:get-inherited-attribute-value-004(., 'CAT')"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
 

</xsl:stylesheet>
