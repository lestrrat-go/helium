<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
                xpath-default-namespace="http://www.examples.com/included"
                version="2.0">

	<xsl:template match="str">
	<xsl:text>&#10;</xsl:text>
	    <str>
	    	<xsl:number count="str" />
	    	<xsl:text></xsl:text>
	    	<xsl:value-of select="'  Matched Str element: Inside included stylesheet'" />
	    </str>
	</xsl:template>


</xsl:stylesheet>
