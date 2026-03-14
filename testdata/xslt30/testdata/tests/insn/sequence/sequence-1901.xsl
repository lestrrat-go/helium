<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Tests xs:anyURI types in a sequence. -->

   <xslt:variable name="uris" as="xs:anyURI*">
      <xslt:sequence select="(xs:untypedAtomic('http://www.hello.com'),doc/uri,xs:anyURI('ftp://ftp.example.com'))"/>
   </xslt:variable>

   <xslt:template match="/">
      <out>
         <xslt:text>
</xslt:text>
         <uri>
            <xslt:value-of select="$uris[0]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$uris[0] instance of xs:anyURI"/>
         </uri>
         <xslt:text>
</xslt:text>
         <uri>
            <xslt:value-of select="$uris[1]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$uris[1] instance of xs:anyURI"/>
         </uri>
         <xslt:text>
</xslt:text>
         <uri>
            <xslt:value-of select="$uris[2]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$uris[2] instance of xs:anyURI"/>
         </uri>
         <xslt:text>
</xslt:text>
         <uri>
            <xslt:value-of select="$uris[3]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$uris[3] instance of xs:anyURI"/>
         </uri>
         <xslt:text>
</xslt:text>
         <uri>
            <xslt:value-of select="$uris[4]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$uris[4] instance of xs:anyURI"/>
         </uri>
         <xslt:text>
</xslt:text>
      </out>
   </xslt:template>
</xslt:transform>
