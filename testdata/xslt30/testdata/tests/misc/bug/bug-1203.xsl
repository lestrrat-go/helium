<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
	<xsl:template match="/" name="main">
		<xsl:variable name="gt" select="'&amp;gt;'" />
		<root>
			<a>
				<xsl:analyze-string regex="." select="'a'">
					<xsl:matching-substring>
						<xsl:value-of disable-output-escaping="yes" select="$gt" />
					</xsl:matching-substring>
				</xsl:analyze-string>
			</a>
			<b>
				<xsl:value-of disable-output-escaping="yes" select="$gt" />
			</b>
		</root>
	</xsl:template>
</xsl:stylesheet>
