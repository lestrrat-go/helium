<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <xsl:output indent="yes" encoding="iso-8859-1"/>

  <!-- trivial test of normalized output (uncommited method, NFKD) -->
  <?spec ser#normalization-form?>
  
  <xsl:output normalization-form="NFKD"/>
  <xsl:template match="/">
  <html>
    <head>
        <title>Normalized HTML output</title>
    </head>
    <body>
      <xsl:apply-templates select="doc/*"/>
    </body>
  </html>  
  </xsl:template>
  
  <xsl:template match="en | de | ffi">
    <p><xsl:copy-of select="@*|node()"/></p>
  </xsl:template>
    

</xsl:stylesheet>
