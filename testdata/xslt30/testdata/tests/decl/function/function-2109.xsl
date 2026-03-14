<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:hcmc="http://hcmc.uvic.ca/ns"
   xmlns:xh="http://www.w3.org/1999/xhtml"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   exclude-result-prefixes="#all"
   version="3.0">
   
   <!-- Saxon bug #5913 -->
   <!-- Tests tail recursive call on a different function, combined with lazy evaluation of variables -->
   
   <xsl:template name="xsl:initial-template">
      <xsl:variable name="testPlaceDiv1" as="element(xh:div)">
         <div xmlns="http://www.w3.org/1999/xhtml">
            <ul class="family">
               <li><span class="name">Abe, Yoshikazu</span></li>
            </ul>
         </div>
      </xsl:variable>
      
      <out>
         <xsl:value-of select="hcmc:getLocationAlphabeticBreakPoints($testPlaceDiv1, 10)"/>
      </out>
   </xsl:template>
   
   <xsl:function name="hcmc:getLocationAlphabeticBreakPoints" as="xs:string+">
      <xsl:param name="placeDiv" as="element(xh:div)"/>
      <xsl:param name="threshold" as="xs:integer"/>
      <xsl:variable name="candidateSequence" as="xs:string+" select="(for $i in string-to-codepoints('A')[1] to string-to-codepoints('Z')[1] return codepoints-to-string($i))"/> 
      
      <xsl:sequence 
         select="hcmc:getNextBreakpoint($placeDiv, $threshold, 0, $candidateSequence, 0, ())"/>
   </xsl:function>
   
   <xsl:function name="hcmc:getNextBreakpoint" as="xs:string+">
      <xsl:param name="placeDiv" as="element(xh:div)"/>
      <xsl:param name="threshold" as="xs:integer"/>
      <xsl:param name="prevPos" as="xs:integer"/>
      <xsl:param name="candidateSequence" as="xs:string+"/>
      <xsl:param name="currTotal" as="xs:integer"/>
      <xsl:param name="sequenceSoFar" as="xs:string*"/>
      
      <xsl:choose>
         <xsl:when test="$prevPos ge (count($candidateSequence) - 1)">
            <xsl:choose>
               <xsl:when test="($currTotal + count($placeDiv//xh:li[xh:span[@class='name'][starts-with(., $candidateSequence[last()])]])) gt $threshold">
                  <xsl:sequence select="($sequenceSoFar, $candidateSequence[last()])"/>
               </xsl:when>
               <xsl:otherwise>            
                  <xsl:sequence select="$sequenceSoFar"/>
               </xsl:otherwise>
            </xsl:choose>
         </xsl:when>
         <xsl:when test="empty($sequenceSoFar) and $prevPos lt 1">
            <xsl:sequence select="hcmc:getNextBreakpoint($placeDiv, $threshold, 1, $candidateSequence, count($placeDiv//xh:li[xh:span[@class='name'][starts-with(., $candidateSequence[1])]]), $candidateSequence[1])"/>
         </xsl:when>
         <xsl:otherwise>
            <xsl:variable name="nextLetter" as="xs:string" select="$candidateSequence[$prevPos + 1]"/>
            <xsl:variable name="totalForNextLetter" as="xs:integer" select="count($placeDiv//xh:li[xh:span[@class='name'][starts-with(., $nextLetter)]])"/>
            <xsl:variable name="newTotal" as="xs:integer" select="$currTotal + $totalForNextLetter"/>
            <xsl:choose>
               <xsl:when test="($newTotal gt $threshold)">
                  <xsl:sequence select="hcmc:getNextBreakpoint($placeDiv, $threshold, $prevPos + 1, $candidateSequence, $totalForNextLetter, (($sequenceSoFar, if ($totalForNextLetter gt 0) then $candidateSequence[($prevPos + 1)] else ())))"/>
               </xsl:when>
               <xsl:otherwise>
                  <!-- Otherwise, we just move up one. -->
                  <xsl:sequence select="hcmc:getNextBreakpoint($placeDiv, $threshold, ($prevPos + 1), $candidateSequence, $newTotal, $sequenceSoFar)"/>
               </xsl:otherwise>
            </xsl:choose>
         </xsl:otherwise>
      </xsl:choose>
   </xsl:function>
   
</xsl:stylesheet>