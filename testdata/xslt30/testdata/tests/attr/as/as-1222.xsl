<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://myexamplefunc.org"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test type of non-tunnel xsl:with-param,  @as="item()*" and @select contains one of:
				-sequence of xs:untypedAtomic from input file, explicitly created built-in primitive 
				type (xs:double), mixed sequence of explicitly created built-in primitive types, 
				attribute() from input file, document-node() from input file, temporary tree, (),
				empty sequence obtained at run time, literal string -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:variable name="temporary-tree">
      <a>
         <b att="one">hello</b>
      </a>
   </xslt:variable>

   <xslt:template match="/doc">
      <out>
         <xslt:apply-templates select="item">
            <xslt:with-param name="par1" select="/doc//text()" as="item()*" tunnel="no"/>
            <xslt:with-param name="par2" select="xs:double('INF')" as="item()*" tunnel="no"/>
            <xslt:with-param name="par3"
               select="(xs:double(3.0),xs:integer(-300),xs:QName('my:local'), xs:dayTimeDuration('PT99.999S'))"
               as="item()*" tunnel="no"/>
            <xslt:with-param name="par4" select="item/@attrib" as="item()*" tunnel="no"/>
            <xslt:with-param name="par5" select="/" as="item()*" tunnel="no"/>
            <xslt:with-param name="par6" select="$temporary-tree" as="item()*" tunnel="no"/>
            <xslt:with-param name="par7" select="()" as="item()*" tunnel="no"/>
            <xslt:with-param name="par8" select="elem" as="item()*" tunnel="no"/>
            <xslt:with-param name="par9" select="'hello'" as="item()*" tunnel="no"/>
         </xslt:apply-templates>
      </out>
   </xslt:template>

   <xslt:template match="item">
      <xslt:param name="par1" select="1.0"/>
      <xslt:param name="par2" select="1.0"/>
      <xslt:param name="par3" select="1.0"/>
      <xslt:param name="par4" select="1"/>
      <xslt:param name="par5" select="1"/>
      <xslt:param name="par6" select="'wrong'"/>
      <xslt:param name="par7" select="'P14D'"/>
      <xslt:param name="par8" select="'P14D'"/>
      <xslt:param name="par9" select="'wrong'"/>
      <par1>
         <xslt:value-of select="$par1"/>
         <xslt:value-of select="$par1 instance of item()*"/>
      </par1>
      <par2>
         <xslt:value-of select="$par2"/>
         <xslt:value-of select="$par2 instance of item()*"/>
      </par2>
      <par3>
         <xslt:value-of select="$par3"/>
         <xslt:value-of select="$par3 instance of item()*"/>
      </par3>
      <par4>
         <xslt:value-of select="$par4"/>
         <xslt:value-of select="$par4 instance of item()*"/>
      </par4>
      <par5>
         <xslt:value-of select="$par5"/>
         <xslt:value-of select="$par5 instance of item()*"/>
      </par5>
      <par6>
         <xslt:value-of select="$par6"/>
         <xslt:value-of select="$par6 instance of item()*"/>
      </par6>
      <par7>
         <xslt:value-of select="$par7"/>
         <xslt:value-of select="$par7 instance of item()*"/>
      </par7>
      <par8>
         <xslt:value-of select="$par8"/>
         <xslt:value-of select="$par8 instance of item()*"/>
      </par8>
      <par9>
         <xslt:value-of select="$par9"/>
         <xslt:value-of select="$par9 instance of item()*"/>
      </par9>
   </xslt:template>
</xslt:transform>
