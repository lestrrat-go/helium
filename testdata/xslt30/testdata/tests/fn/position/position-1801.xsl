<?xml version="1.0" encoding="utf-8"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0">
                
  <xsl:strip-space elements="*"/>              

  <xsl:template match="/">
    <html>
      <body>
        <xsl:apply-templates/>
      </body>
    </html>
  </xsl:template>

  <xsl:template match='LINE[position()&lt;last()]'>
    <P>
      <xsl:apply-templates/>
    </P>
  </xsl:template>
</xsl:stylesheet>