<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test comparisons between subtypes of integer. -->
  <?spec xpath#id-value-comparisons?>
  
  <xsl:template match="/">
    <out>;
      <t><xsl:value-of select="xs:integer('100') = xs:decimal('100.00')"/></t>;
      <t><xsl:value-of select="xs:long('100') = xs:int('100')"/></t>;
      <t><xsl:value-of select="xs:short('-500') &lt; xs:long(+500)"/></t>;      
      <f><xsl:value-of select="xs:short(-500) gt xs:positiveInteger(+500)"/></f>;                                       
    </out> 
  </xsl:template>
  

</xsl:stylesheet>