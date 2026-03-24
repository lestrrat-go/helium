<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:ped="http://ped.com/"
                exclude-result-prefixes="ped">

<xsl:template match="test">
  <out><xsl:copy-of select="doc('')/xsl:stylesheet/ped:test[@attrib='yeha']"/></out>
</xsl:template>

<ped:test attrib="yeha">YEE-HA</ped:test>
<ped:test attrib="haye">Wrong item from sub</ped:test>

</xsl:stylesheet>