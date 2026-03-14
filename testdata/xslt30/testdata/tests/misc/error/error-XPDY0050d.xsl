<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="http://localhost/functions" 
exclude-result-prefixes="f">

  <!-- PURPOSE: Test "treat as" on sequences. -->
  <?spec xpath#errors?><?error XPDY0050?>
  
<xsl:param name="p" select="12"/>

  <xsl:template match="/">
    <out>;
        <!-- treat-as failure: 13th item in list is a string, not a decimal -->
      
      <t><xsl:value-of select="f:f($p)" separator=", "/></t>;
      <t><xsl:value-of select="(f:f($p)) treat as xs:decimal * " separator=", "/></t>
    </out> 
  </xsl:template>
  
  <xsl:function name="f:f">
    <xsl:param name="x"/>
    <xsl:sequence select="if ($x=0) then ('s') else ($x, f:f($x - 1))"/>
  </xsl:function>

</xsl:stylesheet>