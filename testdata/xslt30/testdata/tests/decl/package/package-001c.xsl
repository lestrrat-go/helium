<!-- the simplest possible package: initial mode must be public, XTDE0045 -->
<xsl:package version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    <xsl:mode name="start" />
    
    <xsl:template match="." mode="start">
        <not-ok />
    </xsl:template>

</xsl:package>   
