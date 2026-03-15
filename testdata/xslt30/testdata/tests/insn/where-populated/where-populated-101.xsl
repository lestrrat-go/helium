<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <xsl:variable name="line" as="element()">
    <line task="1">Line</line>
  </xsl:variable>
  
  <xsl:variable name="text" as="node()*">
    Other text
  </xsl:variable>
  
  <xsl:template name="main">
    <html>
      <body>
        <xsl:where-populated>
          <div tabindex="1">
            <xsl:sequence select="$line!@task" />
            <xsl:value-of select="$text!text()[.]" />
          </div>
        </xsl:where-populated>
      </body>
    </html>
  </xsl:template>
  
</xsl:stylesheet>