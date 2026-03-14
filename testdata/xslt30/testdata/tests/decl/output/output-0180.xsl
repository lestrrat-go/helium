<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- Purpose: Test with unsupported value for @encoding, output method is XML. -->

<xsl:output method="xml" encoding="XXX-xx" indent="no" />

<xsl:template match="/">
	<doc>hello</doc>
</xsl:template>


</xsl:stylesheet>
