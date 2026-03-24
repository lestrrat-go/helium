<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test as attribute of xsl:variable. -->
  <?spec xslt#variable-values?>
  
  <xsl:param name="p" select="1"/>
  
  <xsl:template match="/">
    <out>;
        <xsl:variable name="a" as="xs:integer" select="xs:short('123')"/>
        <xsl:variable name="b" as="xs:string" select="string(123)"/>        
        <xsl:variable name="c" as="xs:boolean*" select="boolean(0)"/>        
        <xsl:variable name="d" as="xs:boolean?" select="if ($p=2) then true() else ()"/>
        <xsl:variable name="e" as="xs:string*" select="for $i in 1 to $p return string($i)"/> 
        <t><xsl:value-of select="$a instance of xs:decimal"/></t>;
        <t><xsl:value-of select="$b instance of xs:string"/></t>;
        <t><xsl:value-of select="$c instance of xs:boolean"/></t>;
        <t><xsl:value-of select="empty($d)"/></t>; 
        <t><xsl:value-of select="$e instance of xs:string"/></t>;                                 
    </out> 
  </xsl:template>
  

</xsl:stylesheet>