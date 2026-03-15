<?xml version="1.0"?>
<?spec xslt#character-maps?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns="http://www.w3.org/1999/xhtml">

  <!-- test simple use of a character map with XHTML output method -->
  <!-- Note, this character map will modify characters in the generated meta element.
       Not very desirable but that's what the spec says. -->

  <xsl:character-map name="map01">
    <xsl:output-character character="c" string="[C]"/>
    <xsl:output-character character="x" string="[X]"/>
  </xsl:character-map>
  
  <xsl:output use-character-maps="map01"/>

  <xsl:template match="/">
  <html>
    <head><title>title</title></head>
    <p>The quick brown fox jumped over the lazy dog</p>
  </html>  
  </xsl:template>

</xsl:stylesheet>