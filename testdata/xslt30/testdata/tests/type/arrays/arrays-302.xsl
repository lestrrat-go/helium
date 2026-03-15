<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
   <xsl:variable name="data" as="element()*">
     <a>1</a>
     <b>2</b>
     <c>3</c>
     <d>4</d>
   </xsl:variable>
  
  <xsl:mode on-no-match="shallow-skip"/>
 
 
  <xsl:template name="xsl:initial-template">
    <out>
      <xsl:apply-templates select="array{$data}"/>
    </out>
  </xsl:template>
  
  <xsl:template match="*">
    <element name="{name()}"/>
  </xsl:template>
            
    
</xsl:stylesheet>