<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


<!-- test collection() function -->

<xsl:template match="/">
  <out>
    <xsl:copy-of select="collection('collection-004.xml')"/>
  </out>
</xsl:template>

</xsl:stylesheet>