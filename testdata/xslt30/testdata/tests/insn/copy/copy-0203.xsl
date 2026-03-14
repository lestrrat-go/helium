<?xml version="1.0"?>

<?spec xslt#copy-of?>
<!-- copy an atomic value -->    

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <xsl:template match="/">
    <out>
      <xsl:copy-of select="32"/>
    </out>
  </xsl:template>
</xsl:stylesheet>
