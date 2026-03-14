<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://www.w3.org/xslt30tests/functions" exclude-result-prefixes="map xs err f">


  <xsl:strip-space elements="*"/>



  <!-- Non-streamable filter function - accepts streamed node in second argument -->
  
  <xsl:function name="f:get-inherited-attribute-value-004" as="xs:string?" streamability="filter">
    <xsl:param name="element" as="node()?"/>
    <xsl:param name="attribute-name"/>
    <xsl:sequence select="
      if (empty($element)) then ()
      else if (exists($element/@*[local-name() = $attribute-name])) 
      then string($element/@*[local-name() = $attribute-name])
      else ''"/>
  </xsl:function>
  
  <xsl:template name="main" >
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="/BOOKLIST/BOOKS/ITEM/WEIGHT ! f:get-inherited-attribute-value-004(., ../ITEM/@CAT)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
 

</xsl:stylesheet>
