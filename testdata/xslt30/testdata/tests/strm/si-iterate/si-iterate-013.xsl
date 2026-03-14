<xsl:stylesheet
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
	version="3.0" 
	xmlns:xs="http://www.w3.org/2001/XMLSchema" 
	exclude-result-prefixes="xs">
	
	<xsl:param name="elements-to-copy" as="xs:integer" select="5"/>
	
	<xsl:mode streamable="yes"/>
	
	<xsl:output indent="no"/>

	
	<xsl:template match="/*">
		<xsl:copy>
			<xsl:iterate select="*">
				<xsl:copy-of select="."/>
				<xsl:if test="position() eq $elements-to-copy">
					<xsl:break/>
				</xsl:if>
			</xsl:iterate>
		</xsl:copy>
	</xsl:template>
	
	<xsl:template match="processing-instruction()"><xsl:copy-of select="."/></xsl:template>
	
</xsl:stylesheet>