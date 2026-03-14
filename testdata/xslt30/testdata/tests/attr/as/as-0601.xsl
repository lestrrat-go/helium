<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://myexamplefunc.org"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Test with non-tunnel xsl:with-param where the value in @select is explicitly constructed and of the same type 
  				as the built-in atomic type in @as. Verify the parapmeter is of the type specified in @as.Types tested are:
				xs:string, xs:double, xs:integer, xs:dayTimeDuration, xs:QName -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:template match="/">
      <out>
         <xslt:apply-templates>
	           <xslt:with-param name="par1"
                             select="xs:double(5.4321E-1001)"
                             as="xs:double"
                             tunnel="no"/>
	           <xslt:with-param name="par2"
                             select="xs:double(-0.0E0)"
                             as="xs:double"
                             tunnel="no"/>
	           <xslt:with-param name="par3"
                             select="xs:double('INF')"
                             as="xs:double"
                             tunnel="no"/>
	           <xslt:with-param name="par4" select="xs:integer(43)" as="xs:integer" tunnel="no"/>
	           <xslt:with-param name="par5"
                             select="xs:integer(-300)"
                             as="xs:integer"
                             tunnel="no"/>
	           <xslt:with-param name="par6"
                             select="xs:QName('my:local')"
                             as="xs:QName"
                             tunnel="no"/>
	           <xslt:with-param name="par7"
                             select="xs:dayTimeDuration('PT99.999S')"
                             as="xs:dayTimeDuration"
                             tunnel="no"/>
	           <xslt:with-param name="par8"
                             select="xs:dayTimeDuration('-PT100M')"
                             as="xs:dayTimeDuration"
                             tunnel="no"/>
	           <xslt:with-param name="par9"
                             select="xs:string('hello')"
                             as="xs:string"
                             tunnel="no"/>
         </xslt:apply-templates>
      </out>
   </xslt:template>

   <xslt:template match="item1">
      <xslt:param name="par1" select="1.0"/>
      <par1>
         <xslt:value-of select="$par1"/>
         <xslt:value-of select="$par1 instance of xs:double"/>
      </par1>
   </xslt:template>

   <xslt:template match="item2">
      <xslt:param name="par2" select="1.0"/>
      <par2>
         <xslt:value-of select="abs($par2)"/>
         <xslt:value-of select="$par2 instance of xs:double"/>
      </par2>
   </xslt:template>

   <xslt:template match="item3">
      <xslt:param name="par3" select="1.0"/>
      <par3>
         <xslt:value-of select="$par3"/>
         <xslt:value-of select="$par3 instance of xs:double"/>
      </par3>
   </xslt:template>

   <xslt:template match="item4">
      <xslt:param name="par4" select="1"/>
      <par4>
         <xslt:value-of select="$par4"/>
         <xslt:value-of select="$par4 instance of xs:integer"/>
      </par4>
   </xslt:template>

   <xslt:template match="item5">
      <xslt:param name="par5" select="1"/>
      <par5>
         <xslt:value-of select="$par5"/>
         <xslt:value-of select="$par5 instance of xs:integer"/>
      </par5>
   </xslt:template>

   <xslt:template match="item6">
      <xslt:param name="par6" select="'wrong'"/>
      <par6>
         <xslt:value-of select="$par6"/>
         <xslt:value-of select="$par6 instance of xs:QName"/>
      </par6>
   </xslt:template>

   <xslt:template match="item7">
      <xslt:param name="par7" select="'P14D'"/>
      <par7>
         <xslt:value-of select="$par7"/>
         <xslt:value-of select="$par7 instance of xs:dayTimeDuration"/>
      </par7>
   </xslt:template>

   <xslt:template match="item8">
      <xslt:param name="par8" select="'P14D'"/>
      <par8>
         <xslt:value-of select="$par8"/>
         <xslt:value-of select="$par8 instance of xs:dayTimeDuration"/>
      </par8>
   </xslt:template>

   <xslt:template match="item9">
      <xslt:param name="par9" select="'wrong'"/>
      <par9>
         <xslt:value-of select="$par9"/>
         <xslt:value-of select="$par9 instance of xs:string"/>
      </par9>
   </xslt:template>
</xslt:transform>
