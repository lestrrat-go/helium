<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
   
  <!-- within xsl:source-document, use xsl:for-each-group  -->
  <!-- These tests are mechanically generated from the xsl:for-each tests and do not do any interesting
       grouping, but are designed to take the population in different postures -->
   
  <xsl:template name="feg-001" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each-group group-adjacent="position()" select="account/transaction[@value &lt; 0]">
          <xsl:copy-of select="current-group()"/>
        </xsl:for-each-group>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:for-each-group  with numbering -->
   
  <xsl:template name="feg-002">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each-group group-adjacent="position()" select="account/transaction[@value &lt; 0]">
          <transaction nr="{position()}">
            <xsl:copy-of select="current-group()/@*"/>
          </transaction>
        </xsl:for-each-group>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:for-each-group  with positional selection -->
   
  <xsl:template name="feg-003">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each-group group-adjacent="position()" select="account/transaction[position() &lt; 5]">
          <transaction nr="{position()}">
            <xsl:copy-of select="current-group()/@*"/>
          </transaction>
        </xsl:for-each-group>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- xsl:for-each-group group-adjacent="position()"  over atomized nodes -->
   
  <xsl:template name="feg-004">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each-group group-adjacent="position()" select="subsequence(data(account/transaction/@value), 5, 3)">
          <transaction nr="{position()}">
            <xsl:value-of select="current-group()"/>
          </transaction>
        </xsl:for-each-group>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- crawling selection, inspection body -->
   
  <xsl:template name="feg-005" expand-text="yes">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each-group group-adjacent="position()" select="(//*)[position()=1 to 6]">{position()}:{name()}</xsl:for-each-group>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- crawling selection, inspection body -->
   
  <xsl:template name="feg-006" use-when="false()">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each-group group-adjacent="position()" select="//*">
          <xsl:sequence select="count(current-group()/ancestor-or-self::*)"/>
        </xsl:for-each-group>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use parent axis within xsl:for-each-group  -->
   
  <xsl:template name="feg-008">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each-group group-adjacent="position()" select="account/transaction[position() lt 5]">
          <xsl:sequence select="current-group()/name(..)"/>
        </xsl:for-each-group>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use ancestor axis within xsl:for-each-group  -->
   
  <xsl:template name="feg-009">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each-group group-adjacent="position()" select="subsequence(account/transaction, 1, 4)">
          <xsl:sequence select="current-group()/name(ancestor::*[1])"/>
        </xsl:for-each-group>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use subsequence() within xsl:for-each-group  -->
   
  <xsl:template name="feg-010">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each-group group-adjacent="position()" select="subsequence(account/transaction, 1, 4)">
          <xsl:copy-of select="current-group()"/>
        </xsl:for-each-group>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, select attributes within xsl:for-each-group -->
   
  <xsl:template name="feg-011">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each-group group-adjacent="position()" select="account/transaction/@value">
          <xsl:if test=". >= 0">
            <xsl:sequence select="current-group()/string(.)"/>
          </xsl:if>  
        </xsl:for-each-group>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, iterate over ancestor axis within xsl:for-each-group  -->
   
  <xsl:template name="feg-012">
    <out xmlns="http://loan.shark.com/">
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:for-each-group group-adjacent="position()" select="outermost(//*:extra)">
          <xsl:for-each-group group-adjacent="position()" select="ancestor::*">
            <xsl:copy>
              <xsl:copy-of select="current-group()/@*"/>
            </xsl:copy>
          </xsl:for-each-group>  
        </xsl:for-each-group>
      </xsl:source-document>
    </out>
  </xsl:template>       

       
</xsl:transform>

