<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:fo="http://www.w3.org/XSL/Format/1.0">

  <!-- test xsl:number level="any" when it cannot be optimized, because "count" is not known statically -->
  <xsl:template match="doc">
    <out>
      <xsl:apply-templates/>
    </out>
  </xsl:template>
  
  <xsl:template match="note|apple|banana|custard">
    <fo:block>
       <xsl:number level="any" from="chapter" format="(1) "/>
       <xsl:number level="any" from="doc" format="[1] "/>
       <xsl:apply-templates/>
    </fo:block>
  </xsl:template>
  
  <xsl:strip-space elements="*"/>
 
</xsl:stylesheet>
