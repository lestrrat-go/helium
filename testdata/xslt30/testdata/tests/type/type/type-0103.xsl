<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test constructors for subtypes of integer. -->
<?spec fo#casting-to-derived-types?>
  
  <xsl:template match="/">
    <out>;
      <t><xsl:value-of select="xs:integer('1000000000000000')"/></t>;
      <t><xsl:value-of select="xs:long('10000000000000000')"/></t>;
      <t><xsl:value-of select="xs:int('-100000000')"/></t>;      
      <t><xsl:value-of select="xs:short('-10000')"/></t>;
      <t><xsl:value-of select="xs:byte('100')"/></t>;
      <t><xsl:value-of select="xs:unsignedLong('10000000000000000')"/></t>;
      <t><xsl:value-of select="xs:unsignedInt('+10000000')"/></t>;
      <t><xsl:value-of select="xs:unsignedShort('65535')"/></t>;
      <t><xsl:value-of select="xs:unsignedByte('255')"/></t>;
      <t><xsl:value-of select="xs:nonNegativeInteger('-0')"/></t>;
      <t><xsl:value-of select="xs:nonPositiveInteger('0')"/></t>;   
      <t><xsl:value-of select="xs:negativeInteger('-2')"/></t>;
      <t><xsl:value-of select="xs:positiveInteger('+2')"/></t>;                                          
    </out> 
  </xsl:template>
  

</xsl:stylesheet>