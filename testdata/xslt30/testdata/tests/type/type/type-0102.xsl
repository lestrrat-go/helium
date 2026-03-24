<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://my.uri/"
 exclude-result-prefixes="my xs">

  <!-- PURPOSE: Test "treat as" on sequences. -->
<?spec xpath#id-treat?>
  
<xsl:param name="p" select="12"/>

  <xsl:template match="/">
    <out>;
      <!-- the next two are known at compile time -->
      <t><xsl:value-of select="(23, 24) treat as xs:decimal +"/></t>;
      <t><xsl:value-of select="23 treat as xs:integer ?"/></t>;
      
      <!-- here are some that aren't known until run-time -->
      <t><xsl:value-of select="my:f($p)" separator=", "/></t>;
      <t><xsl:value-of select="my:f($p) treat as xs:decimal * " separator=", "/></t>;
      <t><xsl:value-of select="my:f(0) treat as xs:decimal ? " separator=", "/></t>
    </out> 
  </xsl:template>
  
  <xsl:function name="my:f">
    <xsl:param name="x"/>
    <xsl:sequence select="if ($x=0) then () else ($x, my:f($x - 1))"/>
  </xsl:function>

</xsl:stylesheet>