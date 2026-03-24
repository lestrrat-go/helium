<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" use-when="false()"
version="2.0">

	<xsl:template match="a">
		<print_a>From the included stylesheet. </print_a>
	</xsl:template>
	
	<xsl:template match="b">
		<print_b>From the Included stylesheet</print_b>
	</xsl:template>

</xsl:transform>