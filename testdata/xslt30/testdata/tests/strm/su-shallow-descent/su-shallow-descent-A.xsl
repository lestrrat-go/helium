<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://www.w3.org/xslt30tests/functions" exclude-result-prefixes="map xs err f">
  
  <xsl:variable name="RUN" select="true()" static="yes"/>
  <xsl:strip-space elements="*"/>
  
  
  <!-- Test arity=1 shallow-descent function -->
  
  <xsl:function name="f:alternate-children" as="node()*" 
    streamability="shallow-descent">
    <xsl:param name="input" as="element()"/>
    <xsl:sequence select="$input/*[position() mod 2 = 1]"/>
  </xsl:function>
  
  <xsl:template name="t-001" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:for-each select="f:alternate-children(/BOOKLIST/BOOKS)">
          <xsl:copy-of select="TITLE"/>
        </xsl:for-each>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test arity=1 recursive shallow-descent function -->
  
  <xsl:function name="f:odd-children" as="node()*" 
    streamability="shallow-descent">
    <xsl:param name="input" as="element()"/>
    <xsl:param name="level" as="xs:integer"/>
    <xsl:sequence select="if ($level=0) then $input/*[position() mod 2 = 1] else $input/* ! f:odd-children(., $level - 1)"/>
  </xsl:function>
  
  <xsl:template name="t-002" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:copy-of select="f:odd-children(/BOOKLIST, 2)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <xsl:template name="t-003" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:copy-of select="f:odd-children(copy-of(/BOOKLIST), 2)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Motionless body is allowed -->
  
  <xsl:function name="f:first-born" as="node()*" 
    streamability="shallow-descent">
    <xsl:param name="input" as="element()"/>
    <xsl:variable name="e" as="element()"><xxx/></xsl:variable>
    <xsl:sequence select="if (has-children($input)) then $input else f:first-born($e)"/>
  </xsl:function>
  
  <xsl:template name="t-004" use-when="true() or $RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:copy-of select="f:first-born(/BOOKLIST)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
<!--  
  <!-\- Test arity=2 shallow-descent function -\->
  
  <xsl:function name="f:get-attribute-value-003" as="xs:string"  streamability="shallow-descent" use-when="true() or $RUN">
    <xsl:param name="element" as="node()"/>
    <xsl:param name="attribute-name" as="xs:string"/>
    <xsl:sequence select="string($element/@*[local-name() = $attribute-name])"/>
  </xsl:function>
  
  <xsl:template name="t-003" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="/BOOKLIST/BOOKS/ITEM/WEIGHT ! f:get-attribute-value-003(., 'UNIT')"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-\- Test arity=2 shallow-descent function with recursion -\->
  
  <xsl:function name="f:get-inherited-attribute-value-004"  streamability="shallow-descent" as="xs:string?" use-when="true() or $RUN">
    <xsl:param name="element" as="node()?"/>
    <xsl:param name="attribute-name" as="xs:string"/>
    <xsl:sequence select="
      if (empty($element)) then ()
      else if (exists($element/@*[local-name() = $attribute-name])) 
      then string($element/@*[local-name() = $attribute-name])
      else f:get-inherited-attribute-value-004($element/.., $attribute-name)"/>
  </xsl:function>
  
  <xsl:template name="t-004" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="/BOOKLIST/BOOKS/ITEM/WEIGHT ! f:get-inherited-attribute-value-004(., 'CAT')"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-\- Test arity=1 recursive shallow-descent function returning an ancestor node-\->
  
  <xsl:function name="f:outermost-section" as="element(section)" streamability="shallow-descent">
    <xsl:param name="input" as="node()"/>
    <xsl:sequence select="if ($input/ancestor::section) then f:outermost-section($input/ancestor::section[1]) else $input"/>
  </xsl:function>
  
  <xsl:template name="t-005" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/recursive.xml">
      <out>
        <xsl:value-of select="//foot ! f:outermost-section(.) ! @id"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-\- Test arity=1 shallow-descent function returning an ancestor node-\->
  
  <xsl:function name="f:outermost-section-id"  streamability="shallow-descent">
    <xsl:param name="input" as="node()?"/>
    <xsl:sequence select="if ($input) then if (matches($input/@id, '^[0-9]+\.[0-9]+$')) then $input else f:outermost-section-id($input/..) else ()"/>
  </xsl:function>
  
  <xsl:template name="t-006" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/recursive.xml">
      <out>
        <xsl:value-of select="//foot ! f:outermost-section-id(.) ! @id"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
 
-->
</xsl:stylesheet>
