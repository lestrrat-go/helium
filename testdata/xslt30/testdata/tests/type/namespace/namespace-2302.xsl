<xsl:stylesheet version="2.0" 
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 	
>
	
	<!-- TEST node-name() function (returning a QName) -->
	<?spec fo#func-node-name?>
	
	<xsl:template match="e">
        <z>
        <xsl:for-each select="child::node() | @*">
           <xsl:sort select="namespace-uri()"/>
           <xsl:sort select="local-name()"/>
           <n p='{position()}'
              namespace="{namespace-uri-from-QName(node-name(.))}" 
              local-name="{local-name-from-QName(node-name(.))}"
              prefix="{prefix-from-QName(node-name(.))}"
              content="{.}"/>
        </xsl:for-each>
        </z>
	</xsl:template>

</xsl:stylesheet>
