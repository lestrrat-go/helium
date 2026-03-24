<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://www.mytest.net"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Tests casting different types to xs:string in an xsl:message instruction.  -->

   <xslt:variable name="var1" select="xs:double(5.4321E-1001)" as="xs:double"/>

   <xslt:variable name="var2" select="xs:float(-0.0E0)" as="xs:float"/>

   <xslt:variable name="var3" select="xs:double('INF')" as="xs:double"/>

   <xslt:variable name="var4" select="xs:integer(23)" as="xs:integer"/>

   <xslt:variable name="var5" select="xs:integer(-100)" as="xs:integer"/>

   <xslt:variable name="var6" select="xs:QName('my:var')" as="xs:QName"/>

   <xslt:variable name="var7"
                  select="xs:dayTimeDuration('PT99.999S')"
                  as="xs:dayTimeDuration"/>

   <xslt:variable name="var8"
                  select="xs:dayTimeDuration('-PT100M')"
                  as="xs:dayTimeDuration"/>

   <xslt:variable name="var9" select="xs:string('text')" as="xs:string"/>

   <xslt:variable name="var10" select="xs:anyURI('www.myuri.mine')" as="xs:anyURI"/>

   <xslt:template match="docs">
   		<out>
         <xslt:text>
</xslt:text>
         <xslt:message terminate="no" select="$var1"/>
         <xslt:message select="$var2"/>
         <xslt:message select="$var3"/>
         <xslt:message select="$var4"/>
         <xslt:message select="$var5"/>
         <xslt:message select="$var6"/>
         <xslt:message select="$var7"/>
         <xslt:message select="$var8"/>
         <xslt:message select="$var9"/>
         <xslt:message select="$var10"/>
         <xslt:call-template name="cast"/>
      </out>
	  </xslt:template>

   <xslt:template name="cast">
		    <var1 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <xslt:value-of select="$var1 instance of xs:double"/>
      </var1>
		    <xslt:text>
</xslt:text>
		    <var2 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <xslt:value-of select="$var2 instance of xs:float"/>
      </var2>
		    <xslt:text>
</xslt:text>
		    <var3 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <xslt:value-of select="$var3 instance of xs:double"/>
      </var3>
		    <xslt:text>
</xslt:text>
		    <var4 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <xslt:value-of select="$var4 instance of xs:integer"/>
      </var4>
		    <xslt:text>
</xslt:text>
		    <var5 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <xslt:value-of select="$var5 instance of xs:integer"/>
      </var5>
		    <xslt:text>
</xslt:text>
		    <var6 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <xslt:value-of select="$var6 instance of xs:QName"/>
      </var6>
		    <xslt:text>
</xslt:text>
		    <var7 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <xslt:value-of select="$var7 instance of xs:dayTimeDuration"/>
      </var7>
		    <xslt:text>
</xslt:text>
		    <var8 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <xslt:value-of select="$var8 instance of xs:dayTimeDuration"/>
      </var8>
		    <xslt:text>
</xslt:text>
		    <var9 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <xslt:value-of select="$var9 instance of xs:string"/>
      </var9>
		    <xslt:text>
</xslt:text>
		    <var10 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <xslt:value-of select="$var10 instance of xs:anyURI"/>
      </var10>
		    <xslt:text>
</xslt:text>
	  </xslt:template>
</xslt:transform>
