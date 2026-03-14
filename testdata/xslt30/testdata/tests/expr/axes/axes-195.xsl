<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<?spec dm#TextNode?>
<!-- test that adjacent text nodes are merged -->
	<xsl:template match="/a/b">
	    <xsl:variable name="temp">
	      <a>
	        <xsl:text>123</xsl:text>
	        <xsl:text>456</xsl:text>
	      </a>
	    </xsl:variable>
	    <out>
              <xsl:value-of select="count($temp/a/text())"/>
	    </out>
	</xsl:template>
</xsl:stylesheet>
