<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema" version="2.0">

  <!-- Additional test of unary minus -->
  <!-- See Saxon bug 2859247 -->
  <!-- Submitted by Alex Jitianu, oXygen -->


    <xsl:template name="dataSet">
        <xsl:param name="pointValue" as="xs:integer"></xsl:param>
        <xsl:param name="yScaleFactorRight1" as="xs:integer"></xsl:param>
        
        <xsl:variable name="coordY" select="$pointValue div $yScaleFactorRight1"/>
        
        <circle cx="10" cy="{-$coordY}" r="4"/>
        
    </xsl:template>
    
    <xsl:template match="/">
        <xsl:call-template name="dataSet">
            <xsl:with-param name="pointValue">1</xsl:with-param>
            <xsl:with-param name="yScaleFactorRight1">1</xsl:with-param>
        </xsl:call-template>
    </xsl:template>
</xsl:transform>