<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Tests xs:yearMonthDuration, xs:dayTimeDuration types in a sequence. -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:variable name="daytimes" as="item()*">
      <xslt:sequence select="(xs:untypedAtomic('P4D'),doc/daytime,xs:dayTimeDuration('P4DT7H'))"/>
   </xslt:variable>

   <xslt:variable name="yearmonths" as="item()*">
      <xslt:sequence select="(xs:untypedAtomic('P4Y'),doc/yearmonth,xs:yearMonthDuration('P7Y8M'))"/>
   </xslt:variable>

   <xslt:template match="/">
      <out>
         <xslt:text>
</xslt:text>
         <daytimeout>
            <xslt:value-of select="$daytimes[0]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$daytimes[0] instance of xs:dayTimeDuration"/>
         </daytimeout>
         <xslt:text>
</xslt:text>
         <daytimeout>
            <xslt:value-of select="$daytimes[1]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$daytimes[1] instance of xs:dayTimeDuration"/>
         </daytimeout>
         <xslt:text>
</xslt:text>
         <daytimeout>
            <xslt:value-of select="$daytimes[2]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$daytimes[2] instance of xs:dayTimeDuration"/>
         </daytimeout>
         <xslt:text>
</xslt:text>
         <daytimeout>
            <xslt:value-of select="$daytimes[3]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$daytimes[3] instance of xs:dayTimeDuration"/>
         </daytimeout>
         <xslt:text>
</xslt:text>
         <daytimeout>
            <xslt:value-of select="$daytimes[4]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$daytimes[4] instance of xs:dayTimeDuration"/>
         </daytimeout>
         <xslt:text>
</xslt:text>
         <yearmonthout>
            <xslt:value-of select="$yearmonths[0]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$yearmonths[0] instance of xs:yearMonthDuration"/>
         </yearmonthout>
         <xslt:text>
</xslt:text>
         <yearmonthout>
            <xslt:value-of select="$yearmonths[1]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$yearmonths[1] instance of xs:yearMonthDuration"/>
         </yearmonthout>
         <xslt:text>
</xslt:text>
         <yearmonthout>
            <xslt:value-of select="$yearmonths[2]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$yearmonths[2] instance of xs:yearMonthDuration"/>
         </yearmonthout>
         <xslt:text>
</xslt:text>
         <yearmonthout>
            <xslt:value-of select="$yearmonths[3]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$yearmonths[3] instance of xs:yearMonthDuration"/>
         </yearmonthout>
         <xslt:text>
</xslt:text>
         <yearmonthout>
            <xslt:value-of select="$yearmonths[4]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$yearmonths[4] instance of xs:yearMonthDuration"/>
         </yearmonthout>
         <xslt:text>
</xslt:text>
      </out>
   </xslt:template>
</xslt:transform>
