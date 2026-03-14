<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
	xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" version="3.0">

	<xsl:variable name="data">
		<row Date="2015-01-01"/>
		<row Date="2015-01-02"/>
		<row Date="2015-01-03"/>
		<row Date="2015-01-04"/>
		<row Date="2015-01-05"/>
		<row Date="2015-01-06"/>
	</xsl:variable>

	<xsl:template name="xsl:initial-template">
		<out>
			<xsl:iterate select="$data//row">
				<xsl:variable name="days.left" as="xs:integer">
					<xsl:iterate select="following-sibling::row">
						<xsl:param name="days" select="0"/>
						<xsl:on-completion>
							<xsl:sequence select="$days"/>
						</xsl:on-completion>
						<xsl:next-iteration>
							<xsl:with-param name="days" select="$days + 1"/>
						</xsl:next-iteration>
					</xsl:iterate>
				</xsl:variable>
				<day date="{@Date}" days.left="{$days.left}"/>
			</xsl:iterate>
		</out>
	</xsl:template>

</xsl:stylesheet>
