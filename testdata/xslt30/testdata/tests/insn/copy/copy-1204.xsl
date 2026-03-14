<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

    <!-- Purpose: Make an RTF on the fly and watch for incorrect xml-decl placement. -->

<xsl:output method="xml" encoding="UTF-8" standalone="yes" />

<xsl:template match="/">
  <xsl:variable name="var1"><rtf>abc<in x="yz">def</in>ghi</rtf></xsl:variable>
  <out>
    <xsl:copy-of select="$var1"/>
  </out>
</xsl:template>

</xsl:stylesheet>