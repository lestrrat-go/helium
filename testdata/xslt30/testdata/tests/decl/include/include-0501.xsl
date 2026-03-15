<?xml version="1.0" encoding="iso-8859-1"?>

<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
	<xsl:import href="include-0501a.xsl"/>
	<xsl:import href="include-0501b.xsl"/>
	
	<xsl:template match="/">
		<out>
				<p>
					<xsl:text>First param: </xsl:text>
					<xsl:value-of select="$first"/>
				</p>
				<p>
					<xsl:text>Second param: </xsl:text>
					<xsl:value-of select="$second"/>
				</p>
		</out>
	</xsl:template>
</xsl:stylesheet>