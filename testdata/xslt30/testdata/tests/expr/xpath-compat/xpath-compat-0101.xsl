<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" version="2.0" exclude-result-prefixes="xs">
   <!-- Purpose: Test string rendition of certain numbers that are not backward compatible. -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:template match="/" version="1.0">
      <out>
         <posInf>
            <xslt:value-of select="string(xs:float('INF'))"/>
         </posInf>
         <negInf>
            <xslt:value-of select="string(xs:float('-INF'))"/>
         </negInf>
         <negZero>
            <xslt:value-of select="string(xs:float(-0))"/>
         </negZero>
         <small>
            <xslt:value-of select="string(xs:float(0.00000001))"/>
         </small>
         <large>
            <xslt:value-of select="string(xs:float(10000000))"/>
         </large>
      </out>
   </xslt:template>
</xslt:transform>
