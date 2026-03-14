<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://local-functions.org/"
    exclude-result-prefixes="map xs f">
    
    <xsl:variable name="m" as="map(*)" select="
      map{ 'Document/Options/Updated':'true'  }"/>
    

    <xsl:template name="xsl:initial-template">
      <out>
        <contains><xsl:value-of select="map:contains($m, '')"/></contains>
        <get><xsl:value-of select="map:get($m, '')"/></get>
      </out>
    </xsl:template>
    
 

    
</xsl:stylesheet>