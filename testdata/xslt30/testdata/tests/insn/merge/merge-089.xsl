<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
	xmlns:xs="http://www.w3.org/2001/XMLSchema"
	xmlns:math="http://www.w3.org/2005/xpath-functions/math" exclude-result-prefixes="xs math"
	version="3.0">

	<xsl:output indent="yes"/>

	<xsl:template match="root">
		<xsl:copy>
			<xsl:merge>
				<xsl:merge-source select="item">
					<xsl:merge-key select="foo"/>
				</xsl:merge-source>
				<xsl:merge-action>
					<group key="{current-merge-key()}">
						<xsl:merge>
							<xsl:merge-source select="current-merge-group()">
								<xsl:merge-key select="bar"></xsl:merge-key>
							</xsl:merge-source>
							<xsl:merge-action>
								<nested-group key="{current-merge-key()}">
									<xsl:copy-of select="current-merge-group()/value"/>
								</nested-group>
							</xsl:merge-action>
						</xsl:merge>
					</group>
				</xsl:merge-action>
			</xsl:merge>
		</xsl:copy>
	</xsl:template>

</xsl:stylesheet>
