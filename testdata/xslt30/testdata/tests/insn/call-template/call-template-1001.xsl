<?xml version="1.0"?> 
<?spec xslt#named-templates?>
<xsl:stylesheet 
xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<xsl:output method="xml"/>

<!-- test deep recursion. This version is not tail-recursive; ntmp08 is a variant that is -->

  <xsl:template match="/">
	<xsl:element name="wrap" >
		<xsl:call-template name="reverse" >
		  <xsl:with-param name="index" select="500"/>
		</xsl:call-template>  
	</xsl:element>
  </xsl:template>

<xsl:template name="reverse">
   <xsl:param name="index"/>
    <xsl:if test="$index != 0">
        <xsl:call-template name="reverse">
           <xsl:with-param name="index" select="$index - 1" />
        </xsl:call-template>
        <xsl:value-of select="concat($index,'|')" />
    </xsl:if>
</xsl:template>

<xsl:template match="text()">
   <xsl:call-template name="reverse">
      <xsl:with-param name="index" select="string-length(.)"/>
   </xsl:call-template>
</xsl:template>
</xsl:stylesheet>
