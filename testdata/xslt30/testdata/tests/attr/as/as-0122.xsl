<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://www.mytest.net"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Test with xsl:function that constructs an xs:dayTimeDuration value as a child of LRE and has 
  				@as=xs:dayTimeDuration?. Verify the returned value if of type xs:dayTimeDuration?.-->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:function name="my:func" as="xs:dayTimeDuration?">
	     <elem>
         <xslt:value-of select="xs:dayTimeDuration('PT99.999S')"/>
      </elem>
   </xslt:function>

   <xslt:template match="/doc">
      <out>
         <xslt:value-of select="my:func() instance of xs:dayTimeDuration?"/>
         <xslt:value-of select="my:func() instance of xs:dayTimeDuration"/>
      </out>
   </xslt:template>
</xslt:transform>
