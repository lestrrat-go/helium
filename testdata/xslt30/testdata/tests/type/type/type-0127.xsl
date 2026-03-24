<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://my.com/"
exclude-result-prefixes="xs my">

  <!-- PURPOSE: Test "castable as" operator. -->
  <?spec xpath#id-castable?>
  <?spec fo#casting-boolean?>
  <?spec fo#casting-to-datetimes?>
  
  <xsl:param name="p"/> <!-- force run-time evaluation -->
  
  <xsl:template match="/">
  <out>
    <a><xsl:value-of select="concat('0', $p) castable as xs:boolean"/></a>
    <a><xsl:value-of select="concat('7', $p) castable as xs:boolean"/></a>  
    <a><xsl:value-of select="concat('2000-02-29', $p) castable as xs:date"/></a>      
    <a><xsl:value-of select="concat('2002-02-29', $p) castable as xs:date"/></a>  
    <a><xsl:if test="concat('2000-02-29', $p) castable as xs:date">
          <xsl:value-of select="(concat('2000-02-29',$p)) cast as xs:date"/>
       </xsl:if></a> 
    <a><xsl:choose>
        <xsl:when test="concat('2002-02-29', $p) castable as xs:date">
          <xsl:value-of select="(concat('2002-02-29',$p)) cast as xs:date"/>
       </xsl:when>
       <xsl:otherwise>Invalid date</xsl:otherwise>
       </xsl:choose>
       </a> 
  </out>     
  </xsl:template>                             


</xsl:stylesheet>