<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test "treat as" on simple values. -->
  <?spec xpath#id-treat?>

  
<xsl:param name="p" select="0"/>

  <xsl:template match="/">
    <out>;
      <!-- the next two are known at compile time -->
      <t><xsl:value-of select="23 treat as xs:decimal"/></t>;
      <t><xsl:value-of select="23 treat as xs:integer"/></t>;
      
      <!-- here are some that aren't known until run-time -->
      <t><xsl:value-of select="(if ($p=1) then 's' else 23) treat as xs:decimal "/></t>
      <t><xsl:value-of select="(if ($p=0) then 17.6 else 23) treat as xs:decimal "/></t>      
    </out> 
  </xsl:template>

</xsl:stylesheet>