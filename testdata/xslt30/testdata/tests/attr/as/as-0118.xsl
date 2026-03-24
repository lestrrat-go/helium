<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test type of global xsl:variable where @select contains an xs:dayTimeDuration  
  				or xs:yearMonthDuration, @as="xs:duration".(subtype substitution) -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:variable name="var2"
                  select="xs:dayTimeDuration('PT99.999S')"
                  as="xs:duration"/>

   <xslt:variable name="var3" select="xs:yearMonthDuration('-P21M')" as="xs:duration"/>

   <xslt:template match="/doc">
      <out>       
         <var2>
            <xslt:value-of select="$var2 instance of xs:dayTimeDuration"/>
            <xslt:value-of select="$var2 instance of xs:duration"/>
         </var2>       
         <var3>
            <xslt:value-of select="$var3 instance of xs:yearMonthDuration"/>
            <xslt:value-of select="$var3 instance of xs:duration"/>
         </var3>
      </out>
   </xslt:template>
</xslt:transform>
