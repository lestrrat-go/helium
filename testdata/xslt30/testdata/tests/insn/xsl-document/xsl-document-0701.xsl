<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Tests to sort values inside an xsl:document.  -->

	<xsl:variable name="v" as="document-node()">
		<xsl:document>
			<xsl:for-each select="doc/item">
				<xsl:sort select="@target" lang="en"/>
				<xsl:sort select="@num" lang="en"/>
				<xsl:value-of select="@num"/>
				<xsl:text>:</xsl:text>
				<xsl:value-of select="@target"/>
				<xsl:text> * </xsl:text>
			</xsl:for-each>
		</xsl:document>
	</xsl:variable>

	<xsl:template match="doc">
		<out>
			<xsl:copy-of select="$v"/>
		</out>
	</xsl:template>
</xsl:transform>
