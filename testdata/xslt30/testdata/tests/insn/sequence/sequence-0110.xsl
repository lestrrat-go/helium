<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- PURPOSE:   Test xsl:sequence adding atomic values to an element -->

<xsl:template match="doc">
    <zzz>
      <xsl:sequence select="1 to 5"/>
      <x/>
      <xsl:sequence select="reverse(1 to 5)"/>
    </zzz>
</xsl:template>
 
</xsl:stylesheet>
