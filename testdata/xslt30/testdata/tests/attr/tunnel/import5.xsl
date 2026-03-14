<?xml version="1.0" ?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

	<xsl:template match="last">	
		<xsl:param name="docPar" select="def1" tunnel="yes" />
		<xsl:param name="dataPar" select="def1" tunnel="yes" />
		<xsl:param name="matchPar" select="def1" tunnel="yes" />
		<xsl:param name="inPar" select="def1" tunnel="yes" />
		Value of $docPar is: <xsl:value-of select="$docPar" />
		<xsl:text>&#010;</xsl:text>
		Value of $dataPar is:<xsl:value-of select="$dataPar" />
		<xsl:text>&#010;</xsl:text>
		Value of $matchPar is:<xsl:value-of select="$matchPar" />
		<xsl:text>&#010;</xsl:text>
		Value of $inPar is:<xsl:value-of select="$inPar" />
	</xsl:template>

</xsl:stylesheet>