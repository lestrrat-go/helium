<xsl:stylesheet version="2.0" 
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs"
>
  <!-- test as attribute on xsl:with-param -->
  <?spec xslt#with-param?>
  <?same-as-1.0 no?>
  
  <xsl:template match="/">
   <xsl:apply-templates>
     <xsl:with-param name="arg1" as="node()*">
       <e/><f/><g/>
     </xsl:with-param>
   </xsl:apply-templates>
  </xsl:template>
  
  <xsl:template match="/*">
    <xsl:param name="arg1" as="element()*"/>
    <out count="{count($arg1)}"/>
  </xsl:template>

</xsl:stylesheet>
