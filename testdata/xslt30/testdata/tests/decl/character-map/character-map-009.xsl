<?xml version="1.0"?><?spec xslt#character-maps?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- test character maps with HTML output method. Tests that character maps are not applied
  to HTML attributes where URI-escaping has been applied  -->

  <xsl:output method="html" indent="no" use-character-maps="map02"/>
  
  <xsl:character-map name="map02" xmlns:y="http://charmap/">
    <xsl:output-character character="z" string="[Z]"/>
  </xsl:character-map>
  
  <xsl:template match="/">
    <html>
      <head><title>An HTML page</title></head>
      <xsl:text>&#xa;</xsl:text>
      <body>
        <h1>A heading</h1>
        <a href="z-linkage.html">A z linkage</a>
      </body>
    </html>
  </xsl:template>

</xsl:stylesheet>