<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
   
  <!-- within xsl:source-document, use xsl:iterate -->
  
  <!-- the tests in this stylesheet use xsl:iterate to emulate xsl:for-each -->
   
  <xsl:template name="fe-001" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:iterate select="account/transaction[@value &lt; 0]">
          <xsl:copy-of select="."/>
        </xsl:iterate>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:iterate with numbering -->
   
  <xsl:template name="fe-002" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:iterate select="account/transaction[@value &lt; 0]">
          <transaction nr="{position()}">
            <xsl:copy-of select="@*"/>
          </transaction>
        </xsl:iterate>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:iterate with positional selection -->
   
  <xsl:template name="fe-003" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:iterate select="account/transaction[position() &lt; 5]">
          <transaction nr="{position()}">
            <xsl:copy-of select="@*"/>
          </transaction>
        </xsl:iterate>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- xsl:iterate over atomized nodes -->
   
  <xsl:template name="fe-004" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:iterate select="subsequence(data(account/transaction/@value), 5, 3)">
          <transaction nr="{position()}">
            <xsl:value-of select="."/>
          </transaction>
        </xsl:iterate>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- crawling selection, inspection body -->
   
  <xsl:template name="fe-005" expand-text="yes" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:iterate select="(//*)[position()=1 to 6]">{position()}:{name()} </xsl:iterate>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- crawling selection, inspection body -->
   
  <xsl:template name="fe-006" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:iterate select="//*">
          <xsl:sequence select="count(ancestor-or-self::*)"/>
        </xsl:iterate>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use parent axis within xsl:iterate -->
   
  <xsl:template name="fe-008" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:iterate select="account/transaction[position() lt 5]">
          <xsl:sequence select="name(..)"/>
        </xsl:iterate>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use ancestor axis within xsl:iterate -->
   
  <xsl:template name="fe-009" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:iterate select="subsequence(account/transaction, 1, 4)">
          <xsl:sequence select="name(ancestor::*[1])"/>
        </xsl:iterate>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use subsequence() within xsl:iterate/@select -->
   
  <xsl:template name="fe-010" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:iterate select="subsequence(account/transaction, 1, 4)">
          <xsl:copy-of select="."/>
        </xsl:iterate>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, select attributes within xsl:iterate/@select -->
   
  <xsl:template name="fe-011" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:iterate select="account/transaction/@value">
          <xsl:if test=". >= 0">
            <xsl:sequence select="string(.)"/>
          </xsl:if>  
        </xsl:iterate>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, iterate over ancestor axis within xsl:iterate/@select -->
   
  <xsl:template name="fe-012" use-when="$RUN">
    <out xmlns="http://loan.shark.com/">
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:iterate select="outermost(//*:extra)">
          <xsl:iterate select="ancestor::*">
            <xsl:copy>
              <xsl:copy-of select="@*"/>
            </xsl:copy>
          </xsl:iterate>  
        </xsl:iterate>
      </xsl:source-document>
    </out>
  </xsl:template>       

       
</xsl:transform>

