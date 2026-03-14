<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test xs:unsignedLong value out of range. -->
  <?spec xpath#id-predefined-types?>
  <?error FORG0001 FOAR0002?><!-- (see bug 892) -->
  
  <xsl:param name="p" select="'100000000000000000000000000000000000000000'"/>
  <xsl:template match="/">
    <out>;
      <t><xsl:value-of select="xs:unsignedLong($p) gt 5"/></t>;                              
    </out> 
  </xsl:template>
  

</xsl:stylesheet>