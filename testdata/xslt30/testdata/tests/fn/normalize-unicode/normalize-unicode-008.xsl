<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:f="http://lohalhost/functions"
xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:output indent="yes" encoding="iso-8859-1"/>

  <!-- This test applies the normalize-unicode() function to the
       normalization conformance test published by the Unicode
       Consortium. If successful, no <failure> elements are output.
       For details of the test, see NormalizationTest.txt. The test
       has been modified to avoid using characters that are illegal in XML. -->
       <?spec fo#func-normalize-unicode?>

<xsl:param name="debug" select="false()"/>

<xsl:variable name="datafile" select="unparsed-text('NormalizationTest.txt', 'UTF-8')"/>

<xsl:variable name="lines" 
              select="tokenize($datafile, '\r?\n')
                           [not(starts-with(., '#') or starts-with(., '@'))]"/>

  <xsl:template match="/">
  <out>
   <xsl:for-each select="$lines[position()=1 to 1000]">
     <xsl:analyze-string select="." regex="(.*?);(.*?);(.*?);(.*?);(.*?);(.*)">
       <xsl:matching-substring>
         <xsl:variable name="test-result" 
           select="f:test(regex-group(1), regex-group(2), regex-group(3), regex-group(4), regex-group(5))"/>
         <xsl:if test="count($test-result) != 0">
           <failure code="{$test-result}"><xsl:value-of select="regex-group(6)"/></failure>
         </xsl:if>
       </xsl:matching-substring>
       <xsl:non-matching-substring/>
     </xsl:analyze-string>
   </xsl:for-each>
 </out>
 </xsl:template>
 
 <xsl:function name="f:test" as="xs:string*">
   <xsl:param name="c1raw"/>
   <xsl:param name="c2raw"/>
   <xsl:param name="c3raw"/>
   <xsl:param name="c4raw"/>
   <xsl:param name="c5raw"/>
   <xsl:variable name="c1" select="f:read-hex($c1raw)"/>   
   <xsl:variable name="c2" select="f:read-hex($c2raw)"/>   
   <xsl:variable name="c3" select="f:read-hex($c3raw)"/>   
   <xsl:variable name="c4" select="f:read-hex($c4raw)"/>   
   <xsl:variable name="c5" select="f:read-hex($c5raw)"/>
   <xsl:if test="$debug">
      <xsl:message>c1 = <xsl:value-of select="string-to-codepoints($c1)"/></xsl:message>
      <xsl:message>c2 = <xsl:value-of select="string-to-codepoints($c2)"/></xsl:message>
      <xsl:message>c3 = <xsl:value-of select="string-to-codepoints($c3)"/></xsl:message>
      <xsl:message>c4 = <xsl:value-of select="string-to-codepoints($c4)"/></xsl:message>
      <xsl:message>c5 = <xsl:value-of select="string-to-codepoints($c5)"/></xsl:message>  
   </xsl:if>            
   <xsl:variable name="result" select='
     f:compare($c2, normalize-unicode($c1, "NFC"), "NFC21"),
     f:compare($c2, normalize-unicode($c2, "NFC"), "NFC22"),
     f:compare($c2, normalize-unicode($c3, "NFC"), "NFC23"),
     f:compare($c4, normalize-unicode($c4, "NFC"), "NFC44"),
     f:compare($c4, normalize-unicode($c5, "NFC"), "NFC45"),
     f:compare($c3, normalize-unicode($c1, "NFD"), "NFD31"),
     f:compare($c3, normalize-unicode($c2, "NFD"), "NFD32"),
     f:compare($c3, normalize-unicode($c3, "NFD"), "NFD33"),
     f:compare($c5, normalize-unicode($c4, "NFD"), "NFD54"),
     f:compare($c5, normalize-unicode($c5, "NFD"), "NFD55"),
     f:compare($c4, normalize-unicode($c1, "NFKC"), "NFKC41"),
     f:compare($c4, normalize-unicode($c2, "NFKC"), "NFKC42"),
     f:compare($c4, normalize-unicode($c3, "NFKC"), "NFKC43"),
     f:compare($c4, normalize-unicode($c4, "NFKC"), "NFKC44"),
     f:compare($c4, normalize-unicode($c5, "NFKC"), "NFKC45"),   
     f:compare($c5, normalize-unicode($c1, "NFKD"), "NFKD51"),
     f:compare($c5, normalize-unicode($c2, "NFKD"), "NFKD52"),
     f:compare($c5, normalize-unicode($c3, "NFKD"), "NFKD53"),
     f:compare($c5, normalize-unicode($c4, "NFKD"), "NFKD54"),
     f:compare($c5, normalize-unicode($c5, "NFKD"), "NFKD55")'/> 
   <xsl:sequence select='if (empty($result)) then () else
     ($result, 
     for $f in ("NFC", "NFD", "NFKC", "NFKD"), $i in 1 to 5 return
     (concat($f, $i),      
     string-join(for $c in string-to-codepoints(normalize-unicode(($c1,$c2,$c3,$c4,$c5)[$i], $f)) 
     return string($c), "-")))'/>                          
 </xsl:function>
 
 <xsl:function name="f:compare">
   <xsl:param name="a1"/>
   <xsl:param name="a2"/>
   <xsl:param name="code"/>
   <xsl:if test="$debug">
     <xsl:message>normalized: <xsl:value-of select="string-to-codepoints($a2)"/></xsl:message>
   </xsl:if>
   <xsl:if test="not(codepoint-equal($a1, $a2))">
     <xsl:sequence select="$code"/>
   </xsl:if>
 </xsl:function>
 
 <xsl:function name="f:read-hex">
   <xsl:param name="in"/>  <!-- e.g. 0055 030C -->
   <xsl:variable name="toks" select="tokenize($in, ' ')"/>
   <xsl:variable name="chars" select="for $t in $toks return f:hex-to-char($t)"/>
   <xsl:variable name="vchars" select="for $t in $chars return 
                         if (f:valid-character($t)) then $t else 32"/>   
   <xsl:sequence select="codepoints-to-string($vchars)"/>
 </xsl:function>
 
 <xsl:function name="f:hex-to-char" as="xs:integer">
   <xsl:param name="in"/> <!-- e.g. 030C -->
   <xsl:sequence select="
   if (string-length($in) eq 1)
      then f:hex-digit-to-integer($in)
      else 16*f:hex-to-char(substring($in, 1, string-length($in)-1)) + 
             f:hex-digit-to-integer(substring($in, string-length($in)))"/>
 </xsl:function>
 
 <xsl:function name="f:hex-digit-to-integer" as="xs:integer">
   <xsl:param name="char"/>
   <xsl:sequence select="string-length(substring-before('0123456789ABCDEF', $char))"/>
 </xsl:function> 
 
 <xsl:function name="f:valid-character" as="xs:boolean">
   <!-- test a character for validity in XML 1.0 -->
   <xsl:param name="c" as="xs:integer"/>
   <xsl:sequence select="$c=9 or $c=10 or $c=13 or $c=(32 to 55295) or $c=(57344 to 65533) or $c=(65536 to 1114111)"/>
 </xsl:function>
 
 <xsl:template name="main">
   <xsl:value-of select="f:hex-to-char('100')"/>
 </xsl:template>
                                    
</xsl:stylesheet>