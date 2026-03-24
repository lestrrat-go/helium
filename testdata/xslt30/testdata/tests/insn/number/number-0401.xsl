<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:fo="http://www.w3.org/XSL/Format/1.0">

  <!-- Example from draft -->
  
  <xsl:strip-space elements="*"/>
  
  <xsl:template match="doc">
    <out>
      <xsl:apply-templates/>
    </out>
  </xsl:template>
  
  <xsl:template match="note">
    <fo:block>
       <xsl:number level="any" from="chapter" format="(1) "/>
       <xsl:apply-templates/>
    </fo:block>
  </xsl:template>
 
</xsl:stylesheet>
