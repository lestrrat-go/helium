<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">

           <!--A dynamic error 
               #at one time, but no longer, occurred#
               if the set of keys in a map produced by the
               evaluation of any expression or instruction contains an instance of any 
               of the types xs:dateTime, xs:date, xs:time, xs:gYear
               xs:gYearMonth, xs:gMonth, xs:gMonthDay, or xs:gDay having a timezone and
               also contains an instance of any of these types having no timezone.-->
               
   <xsl:template name="main">
      <out>
         <xsl:variable name="v" as="map(*)">
         <xsl:map>
            <xsl:map-entry key="xs:date('2014-08-25')" select="0"/>
            <xsl:map-entry key="xs:date('2014-08-26Z')" select="1"/>
         </xsl:map>
         </xsl:variable>
         <xsl:value-of select="map:size($v)" xmlns:map="http://www.w3.org/2005/xpath-functions/map"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
