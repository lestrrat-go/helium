<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <!-- trivial test of normalized output (html, NFKD) -->
  <?spec ser#normalization-form?>

  <xsl:output method="html" normalization-form="NFKD" encoding="utf-16" indent="no"/>
  <xsl:template match="/">
    <html>
      <head>
        <title>Normalised text</title>
      </head>
      <body>
        <xsl:apply-templates select="/doc/*"/>
      </body>
    </html>
  </xsl:template>


  <xsl:template match="en|de|ffi">
    <p class="{name()}">
      <xsl:for-each select="@* except @norm">
        <span class="{name()}">
          <xsl:value-of select="."/>
        </span>
      </xsl:for-each>
      <xsl:value-of select="."/>
    </p>
  </xsl:template>


</xsl:stylesheet>
