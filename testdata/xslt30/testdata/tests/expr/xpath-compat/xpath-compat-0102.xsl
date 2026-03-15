<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" version="2.0" exclude-result-prefixes="xs">
   <!-- Purpose: Test fn:dateTime() being given a sequence of values, of the proper type, for either argument. -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:variable name="dates"
      select="(xs:date('1999-11-11'),xs:date('2006-05-11'),xs:date('1776-07-04'))"/>

   <xslt:variable name="times"
      select="(xs:time('01:02:03'),xs:time('22:22:22'),xs:time('12:34:56'))"/>

   <xslt:template match="/" version="1.0">
      <out>
         <first>
            <xslt:value-of select="dateTime($dates,xs:time('12:34:56'))"/>
         </first>
         <second>
            <xslt:value-of select="dateTime($dates,(xs:time('07:00:11'),xs:time('12:34:56')))"/>
         </second>
         <third>
            <xslt:value-of select="dateTime(xs:date('2006-05-11'),$times)"/>
         </third>
         <fourth>
            <xslt:value-of select="dateTime((xs:date('1888-08-08'),xs:date('1776-07-04')),$times)"/>
         </fourth>
         <fifth>
            <xslt:value-of select="dateTime($dates,$times)"/>
         </fifth>
      </out>
   </xslt:template>
</xslt:transform>
