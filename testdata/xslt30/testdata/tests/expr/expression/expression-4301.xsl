<?xml version="1.0"?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:map="http://www.w3.org/2005/xpath-functions/map"
   exclude-result-prefixes="xs map">
   
   <!-- General comparison for QNames. Saxon-JS bug 3216 -->
   <xsl:param name="blahParam" select="'boo'"/>
   
   <xsl:template name="main">
      <xsl:variable name="blah" select="xs:QName($blahParam)"/>
      <xsl:variable name="bindings" select="map{'blahQ': $blah}"/>
      <out>
         <one><xsl:sequence select="$blah = (xs:QName('blah1'), xs:QName('blah2'),
            xs:QName('blah3'))"/></one>
         <two><xsl:sequence select="$blah = (xs:QName('blah1'), xs:QName('blah2'),
            xs:QName('blah'))"/></two>
         <three><xsl:sequence select="map:get($bindings,'blahQ') = xs:QName('blah')"/></three>
      </out>
   </xsl:template>
   
</xsl:stylesheet>