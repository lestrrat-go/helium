<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test with an xs:untypedAtomic value as a child of LRE in the sequence constructor of a local xsl:variable and @as=xs:dayTimeDuration*. 
  				Verify the variable is of type xs:dayTimeDuration*.   -->

   <xslt:template match="/doc">
      <xslt:variable name="var1" as="xs:dayTimeDuration*">
         <item>
            <xslt:value-of select="item8"/>
         </item>
      </xslt:variable>
      <out>
         <xslt:value-of select="$var1 instance of xs:dayTimeDuration*"/>
         <xslt:value-of select="$var1 instance of xs:dayTimeDuration"/>
      </out>
   </xslt:template>
</xslt:transform>
