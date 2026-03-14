<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
 
  <!-- trivial test of normalized output (XHTML, result-document, NFC) -->
  <?spec ser#normalization-form?>
  
  <xsl:template match="/">
  <xsl:result-document method="xhtml" normalization-form="NFC">
  <html>
    <body>
       <xsl:apply-templates select="*"/>
    </body>
  </html>
  </xsl:result-document>  
  </xsl:template>
  
  <xsl:template match="*">
    <div class="{name()}">
      <xsl:copy-of select="@*"/>
      <xsl:apply-templates/>
    </div>
  </xsl:template>
    

</xsl:stylesheet>
