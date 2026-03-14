<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet exclude-result-prefixes="#all" version="2.0"
	xmlns:testns="426EA502-B148-4ff6-BD1A-3447FB49C511" xmlns:xs="http://www.w3.org/2001/XMLSchema"
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
	<xsl:template match="*">
		<xsl:copy><xsl:apply-templates select="*"/></xsl:copy>
	</xsl:template>
	<xsl:template as="comment()" match="foo[preceding-sibling::foo[. eq current()]]"
		name="testns:test-template">
		<xsl:comment select="'match'" />
	</xsl:template>
</xsl:stylesheet>
