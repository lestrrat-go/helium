<?xml version='1.0'?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:abc="http://abc.uri/">

<!-- Test import precedence for xsl:strip-space -->

<xsl:import href="strip-space-020a.xsl"/>
<xsl:strip-space elements="abc:*"/>
<xsl:preserve-space elements="abc:w"/>

<xsl:template match="/"><xsl:copy-of select="."/></xsl:template>

</xsl:stylesheet>
