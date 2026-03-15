<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" 
exclude-result-prefixes="xs">

  <!-- PURPOSE: Test "instance of" on untyped nodes. -->
  <?spec xpath#id-instance-of?>
  <?spec xpath#id-element-test?>
  <?spec xpath#id-attribute-test?>
  
  <xsl:param name="p" select="'example-uri'"/> <!-- force run-time evaluation -->
  
  <xsl:template match="/">
  <out>
    <xsl:variable name="e" select="doc"/>
    <xsl:variable name="a" select="doc/*/@at"/>    
    <e><xsl:value-of select="$e instance of element()"/></e>
    <e><xsl:value-of select="$e instance of element(*)"/></e>  
    <e><xsl:value-of select="$e instance of element(*, xs:anyType)"/></e>
    <e><xsl:value-of select="$e instance of element(*, xs:untyped)"/></e>
    <e><xsl:value-of select="$e instance of element(*, xs:untypedAtomic)"/></e>  
    <e><xsl:value-of select="$e instance of element(*, xs:anyAtomicType)"/></e>  
    <a><xsl:value-of select="$a instance of attribute()"/></a>
    <a><xsl:value-of select="$a instance of attribute(*)"/></a>  
    <a><xsl:value-of select="$a instance of attribute(*, xs:untypedAtomic)"/></a>  
    <a><xsl:value-of select="$a instance of attribute(*, xs:anyAtomicType)"/></a>                        
    
  </out>     
  </xsl:template>                             


</xsl:stylesheet>