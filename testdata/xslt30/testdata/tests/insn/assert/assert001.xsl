<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
				version="3.0">

    <xsl:param name="p" select="2"/>
    <xsl:param name="disabled" static="yes" select="false()"/>
    <xsl:param name="enabled" static="yes" select="true()"/>
    
	<xsl:template name="go001">
   		<out>
   			<xsl:assert test="$p eq 2">This message means: Test has failed</xsl:assert>
   		</out>
	</xsl:template>
	
	<xsl:template name="go002">
   		<out>
   			<xsl:assert test="$p eq 3">This message means: Test has succeeded</xsl:assert>
   		</out>
	</xsl:template>
	
	<xsl:template name="go003">
   		<out>
   			<xsl:assert use-when="$disabled" test="$p eq 3">This message means: Test has failed</xsl:assert>
   		</out>
	</xsl:template>
	
	<xsl:template name="go004">
   		<out>
   			<xsl:assert use-when="$enabled" test="$p eq 3">This message means: Test has succeeded</xsl:assert>
   		</out>
	</xsl:template>
	
	<xsl:template name="go005">
	    <xsl:try>
   		  <out>
   			<xsl:assert use-when="$enabled" test="$p eq 3">This message can be ignored</xsl:assert>
   		  </out>
   		  <xsl:catch errors="*:XTMM9001">
   		    <out/>
   		  </xsl:catch>
   		</xsl:try> 
	</xsl:template>
	
	<xsl:template name="go006">
   		<out>
   			<xsl:assert error-code="my:ABCD9999" xmlns:my="http://example.com/my"
   			      test="$p eq 3">This message means: Test has succeeded</xsl:assert>
   		</out>
	</xsl:template>
	
	<xsl:variable name="message008">This message means: Test has failed</xsl:variable>
	
	<xsl:template name="go008">
   		<out>
   			<xsl:assert test="$p eq 2" select="$message008"/>
   		</out>
	</xsl:template>
	
	<xsl:template name="go010">
		<out>
			<xsl:try>
				<!-- When an xsl:assert fails, following-sibling instructions are not evaluated-->
				<xsl:text>A</xsl:text>
				<xsl:assert test="$p eq 0" select="$message008"/>
				<xsl:value-of select="10000 div $p"/>
				<xsl:catch errors="*:XTMM9001">B</xsl:catch>	
			</xsl:try>
			<xsl:text>C</xsl:text>
		</out>
	</xsl:template>

 
</xsl:stylesheet>
