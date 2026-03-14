<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:txt="http://www.host.com/txt"
                exclude-result-prefixes="txt">

  <!-- Purpose: Variable name is a QName. -->
 
<xsl:variable name="txt:me" select="'Tommy'"/>

<xsl:template match="/">
 <out>
  <xsl:value-of select="$txt:me"/>
 </out>
</xsl:template>

</xsl:stylesheet>
