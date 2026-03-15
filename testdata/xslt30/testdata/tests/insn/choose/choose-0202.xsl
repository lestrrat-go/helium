<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0"
				xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
	exclude-result-prefixes="#all">

	<xsl:variable name="input">
		<root>
			<Cus TradeCredit="0.0000"
				DaysDelay="5"
				SaleLock="0" />
		</root>
	</xsl:variable>

	<xsl:template name="xsl:initial-template">
		<message>
			<xsl:call-template name="Cus" />
		</message>
	</xsl:template>

	<xsl:template name="Cus">
		<xsl:for-each select="$input/root/Cus">
			<xsl:element name="client">
				<merchant-limit due-limit="{@TradeCredit}" days-limit="{@DaysDelay}">
					<xsl:choose>
						<xsl:when test="@SaleLock = 0">
							<xsl:attribute name="block-strategy" select="'A'" />
						</xsl:when>
						<xsl:when test="@SaleLock = 1">
							<xsl:attribute name="block-strategy" select="'B'" />
						</xsl:when>
						<xsl:when test="@SaleLock = 2">
							<xsl:attribute name="block-strategy" select="'C'" />
						</xsl:when>
						<xsl:when test="@SaleLock = 3">
							<xsl:attribute name="block-strategy" select="'D'" />
						</xsl:when>
						<xsl:when test="@SaleLock = 4">
							<xsl:attribute name="block-strategy" select="'E'" />
						</xsl:when>
						<xsl:when test="@SaleLock = 5">
							<xsl:attribute name="block-strategy" select="'F'" />
						</xsl:when>
					</xsl:choose>
				</merchant-limit>

			</xsl:element>

		</xsl:for-each>
	</xsl:template>

</xsl:stylesheet>