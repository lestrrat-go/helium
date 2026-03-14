<?xml version="1.0" encoding="utf-8"?>

<xsl:stylesheet version="2.0"  
xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    
<!-- Saxon 8.5.1 bug report from Krzysztof Nowicki. Copying a document node followed by an element node
     in the sequence constructor of a new document node. -->

	<xsl:template name="test">
		<xsl:param name="count" select="10" />
		<xsl:param name="result" select="()"/>

		<xsl:choose>
			<xsl:when test="$count &gt; 0">

				<xsl:call-template name="test">
					<xsl:with-param name="count" select="$count - 1" />
					<xsl:with-param name="result">

						<!-- This is the place where the list is extended -->
						<!-- by one element. -->
						<xsl:copy-of select="$result" />
						<test1><xsl:value-of select="$count" /></test1>

					</xsl:with-param>
				</xsl:call-template>
			</xsl:when>
			<xsl:otherwise>
				<testtest>
					<xsl:copy-of select="$result" />
				</testtest>
			</xsl:otherwise>
		</xsl:choose>
	</xsl:template>

	<xsl:template match="/">
		<xsl:call-template name="test" />
	</xsl:template>
	
</xsl:stylesheet>
