<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs"
    expand-text="true">
    
    <!-- Mix values with and without timezone. See bug 28632, overturning 25375 -->
  
    <!-- map-010 in its uncorrected form, using a reserved namespace -->
    
    <xsl:variable name="Z" select="current-dateTime()"/>
    <xsl:variable name="A" select="adjust-dateTime-to-timezone($Z, ())"/>
    
    <xsl:template name="main">
      <out>
         <xsl:variable name="M" select="map{$A :1, $Z :1}"/>
         <a>{map:size($M)}</a>
         <xsl:variable name="M" select="map:put($M, $A, 2)"/>
         <b>{map:size($M)}:{map:get($M, $A)}</b>
         <xsl:variable name="M" select="map:put($M, $Z, 2)"/>
         <c>{map:size($M)}:{map:get($M, $Z)}</c>
         <xsl:variable name="M" select="map:remove($M, $A)"/>
         <d>{map:size($M)}:{map:get($M, $A)}</d>
         <xsl:variable name="M" select="map:remove($M, $Z)"/>
         <e>{map:size($M)}:{map:get($M, $Z)}</e>
      </out>  
    </xsl:template>
    
    <xsl:function name="map:show" as="xs:string" expand-text="false">
      <xsl:param name="map" as="map(*)"/>
      <xsl:value-of>
        <xsl:text>map{</xsl:text>
        <xsl:value-of select="map:for-each($map, function($k, $v) { $k || ' : ' || $v || ', ' })"/>
        <xsl:text>}</xsl:text>
      </xsl:value-of>
    </xsl:function>
    
</xsl:stylesheet>