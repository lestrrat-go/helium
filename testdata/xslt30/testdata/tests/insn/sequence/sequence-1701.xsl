<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Tests xs:date, xs:time, and xs:dateTime types in a sequence. -->

   <xslt:variable name="dates" as="item()*">
      <xslt:sequence select="(xs:untypedAtomic('2004-11-22Z'),doc/date,xs:date('2004-11-24'))"/>
   </xslt:variable>

   <xslt:variable name="times" as="item()*">
      <xslt:sequence select="(xs:untypedAtomic('12:11:10Z'),doc/time,xs:time('01:02:03'))"/>
   </xslt:variable>

   <xslt:variable name="datetimes" as="item()*">
      <xslt:sequence select="(xs:untypedAtomic('2004-11-22T12:11:10Z'),doc/datetime,xs:dateTime('2004-11-24T01:02:03'))"/>
   </xslt:variable>

   <xslt:template match="/">
      <out>
         <xslt:text>
</xslt:text>
         <date>
            <xslt:value-of select="$dates[0]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$dates[0] instance of xs:date"/>
         </date>
         <xslt:text>
</xslt:text>
         <date>
            <xslt:value-of select="$dates[1]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$dates[1] instance of xs:date"/>
         </date>
         <xslt:text>
</xslt:text>
         <date>
            <xslt:value-of select="$dates[2]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$dates[2] instance of xs:date"/>
         </date>
         <xslt:text>
</xslt:text>
         <date>
            <xslt:value-of select="$dates[3]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$dates[3] instance of xs:date"/>
         </date>
         <xslt:text>
</xslt:text>
         <date>
            <xslt:value-of select="$dates[4]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$dates[4] instance of xs:date"/>
         </date>
         <xslt:text>
</xslt:text>
         <xslt:text>
</xslt:text>
         <time>
            <xslt:value-of select="$times[0]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$times[0] instance of xs:time"/>
         </time>
         <xslt:text>
</xslt:text>
         <time>
            <xslt:value-of select="$times[1]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$times[1] instance of xs:time"/>
         </time>
         <xslt:text>
</xslt:text>
         <time>
            <xslt:value-of select="$times[2]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$times[2] instance of xs:time"/>
         </time>
         <xslt:text>
</xslt:text>
         <time>
            <xslt:value-of select="$times[3]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$times[3] instance of xs:time"/>
         </time>
         <xslt:text>
</xslt:text>
         <time>
            <xslt:value-of select="$times[4]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$times[4] instance of xs:time"/>
         </time>
         <xslt:text>
</xslt:text>
         <xslt:text>
</xslt:text>
         <datetime>
            <xslt:value-of select="$datetimes[0]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$datetimes[0] instance of xs:dateTime"/>
         </datetime>
         <xslt:text>
</xslt:text>
         <datetime>
            <xslt:value-of select="$datetimes[1]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$datetimes[1] instance of xs:dateTime"/>
         </datetime>
         <xslt:text>
</xslt:text>
         <datetime>
            <xslt:value-of select="$datetimes[2]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$datetimes[2] instance of xs:dateTime"/>
         </datetime>
         <xslt:text>
</xslt:text>
         <datetime>
            <xslt:value-of select="$datetimes[3]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$datetimes[3] instance of xs:dateTime"/>
         </datetime>
         <xslt:text>
</xslt:text>
         <datetime>
            <xslt:value-of select="$datetimes[4]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$datetimes[4] instance of xs:dateTime"/>
         </datetime>
         <xslt:text>
</xslt:text>
      </out>  
   </xslt:template>
</xslt:transform>
